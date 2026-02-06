package authn_test

// func startTestServer(mux *http.ServeMux) (*http.Server, error) {
// 	var err error
// 	httpServer := &http.Server{
// 		Addr: testAddress,
// 		// ReadTimeout:  5 * time.Minute, // 5 min to allow for delays when testing
// 		// WriteTimeout: 10 * time.Second,
// 		// Handler:   srv.router,
// 		TLSConfig: serverTLSConf,
// 		Handler:   mux,
// 		//ErrorLog:  log.Default(),
// 	}
// 	go func() {
// 		err = httpServer.ListenAndServeTLS("", "")
// 	}()
// 	// Catch any startup errors
// 	time.Sleep(100 * time.Millisecond)
// 	return httpServer, err
// }

// TestMain runs a http server
// Used for all test cases in this package
// func TestMain(m *testing.M) {
// 	utils.SetLogging("info", "")
// 	slog.Info("------ TestMain of httpauthhandler ------")
// 	testAddress = "127.0.0.1:9888"
// 	// hostnames := []string{testAddress}

// 	authBundle = selfsigned.CreateTestCertBundle(TestKeyType)

// 	caCertPool := x509.NewCertPool()
// 	caCertPool.AddCert(authBundle.CaCert)

// 	// serverTLSCert := testenv.X509ToTLS(certsclient.ServerCert, nil)
// 	serverTLSConf = &tls.Config{
// 		Certificates:       []tls.Certificate{*authBundle.ServerCert},
// 		ClientAuth:         tls.VerifyClientCertIfGiven,
// 		ClientCAs:          caCertPool,
// 		MinVersion:         tls.VersionTLS12,
// 		InsecureSkipVerify: false,
// 	}

// 	res := m.Run()

// 	time.Sleep(time.Second)
// 	os.Exit(res)
// }

// // Test certificate based authentication
// func TestAuthClientCert(t *testing.T) {
// 	path1 := "/test1"
// 	path1Hit := 0

// 	// setup server and client environment
// 	mux := http.NewServeMux()
// 	srv, err := startTestServer(mux)
// 	assert.NoError(t, err)
// 	//
// 	mux.HandleFunc(path1, func(http.ResponseWriter, *http.Request) {
// 		slog.Info("TestAuthClientCert: path1 hit")
// 		path1Hit++
// 	})
// 	//
// 	cl := tlsclient.NewTLSClient(testAddress, authBundle.ClientCert, authBundle.CaCert, 0)
// 	assert.NoError(t, err)

// 	clientCert := cl.GetClientCertificate()
// 	assert.NotNil(t, clientCert)

// 	// verify service certificate against CA
// 	caCertPool := x509.NewCertPool()
// 	caCertPool.AddCert(authBundle.CaCert)
// 	opts := x509.VerifyOptions{
// 		Roots:     caCertPool,
// 		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
// 	}
// 	cert, err := x509.ParseCertificate(clientCert.Certificate[0])
// 	if err == nil {
// 		_, err = cert.Verify(opts)
// 	}
// 	assert.NoError(t, err)

// 	//
// 	_, _, err = cl.Get(path1)
// 	assert.NoError(t, err)
// 	_, _, err = cl.Post(path1, nil)
// 	assert.NoError(t, err)
// 	_, _, err = cl.Put(path1, nil)
// 	assert.NoError(t, err)
// 	_, _, err = cl.Delete(path1)
// 	assert.NoError(t, err)
// 	_, _, err = cl.Patch(path1, nil)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 5, path1Hit)

// 	cl.Close()
// 	_ = srv.Close()
// }

// func TestNoClientCert(t *testing.T) {
// 	cl := tlsclient.NewTLSClient(testAddress, nil, authBundle.CaCert, 3)
// 	cl.Close()
// }

// func TestBadClientCert(t *testing.T) {
// 	// use cert not signed by the CA
// 	otherCA, otherPrivKey, _, err := selfsigned.CreateSelfSignedCA(
// 		"", "", "", "", "", 1, TestKeyType)
// 	otherCert, err := selfsigned.CreateClientCert("name", "ou", 1,
// 		authBundle.ClientPrivKey, otherCA, otherPrivKey)
// 	otherTLS := x509CertToTLS(otherCert, authBundle.ClientPrivKey)
// 	assert.NoError(t, err)

// 	cl := tlsclient.NewTLSClient(testAddress, otherTLS, authBundle.CaCert, 0)
// 	// this should produce an error in the log
// 	//assert.Error(t, err)
// 	cl.Close()
// }

// func TestAuthJWT(t *testing.T) {
// 	pathLogin1 := "/login"
// 	pathLogin2 := "/login2"
// 	path3 := "/test3"
// 	path3Hit := 0
// 	user1 := "user1"
// 	password1 := "password1"
// 	secret := make([]byte, 64)
// 	_, _ = rand.Read(secret)

// 	// setup server and client environment
// 	mux := http.NewServeMux()
// 	// Handle a jwt login
// 	mux.HandleFunc(pathLogin1, func(resp http.ResponseWriter, req *http.Request) {
// 		// Is the login API a transport feature? Look into the WoT specification.
// 		authMsg := authnserver.UserLoginArgs{}
// 		slog.Info("TestAuthJWT: login")
// 		body, err := io.ReadAll(req.Body)
// 		require.NoError(t, err)
// 		err = json.Unmarshal(body, &authMsg)

// 		// expect a correlationID
// 		//msgID := req.Header.Get(tlsclient.HTTPCorrelationIDHeader)
// 		assert.NoError(t, err)
// 		assert.Equal(t, user1, authMsg.UserName)
// 		assert.Equal(t, password1, authMsg.Password)

// 		if authMsg.UserName == user1 {
// 			claims := jwt.RegisteredClaims{
// 				ID:      user1,
// 				Issuer:  "me",
// 				Subject: "accessToken",
// 				// In JWT, the expiry time is expressed as unix milliseconds
// 				IssuedAt:  jwt.NewNumericDate(time.Now()),
// 				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second)),
// 			}
// 			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
// 			newToken, err := token.SignedString(secret)
// 			assert.NoError(t, err)
// 			//resp.Header().Set(tlsclient.HTTPCorrelationIDHeader, msgID)
// 			data, _ := json.Marshal(newToken)
// 			_, _ = resp.Write(data)
// 		} else {
// 			// write nothing
// 			_ = err
// 		}
// 		path3Hit++
// 	})
// 	// a second login function that returns nothing
// 	mux.HandleFunc(pathLogin2, func(resp http.ResponseWriter, req *http.Request) {
// 	})

// 	mux.HandleFunc(path3, func(http.ResponseWriter, *http.Request) {
// 		slog.Info("TestAuthJWT: path3 hit")
// 		path3Hit++
// 	})
// 	srv, err := startTestServer(mux)
// 	assert.NoError(t, err)
// 	// fixme, remove dependency on authn
// 	loginMessage := authnserver.UserLoginArgs{
// 		UserName: user1,
// 		Password: password1,
// 	}
// 	cl := tlsclient.NewTLSClient(testAddress, nil, authBundle.CaCert, 0)
// 	jsonArgs, _ := json.Marshal(loginMessage)
// 	resp, _, err := cl.Post(pathLogin1, jsonArgs)
// 	require.NoError(t, err)
// 	newToken := ""
// 	err = json.Unmarshal(resp, &newToken)

// 	// reconnect using the given token

// 	cl.ConnectWithToken(user1, newToken)
// 	_, _, err = cl.Get(path3)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 2, path3Hit)

// 	cl.Close()
// 	_ = srv.Close()
// }

// func TestAuthJWTFail(t *testing.T) {
// 	pathHello1 := "/hello"
// 	clientID := "user1"

// 	// setup server and client environment
// 	mux := http.NewServeMux()
// 	srv, err := startTestServer(mux)
// 	assert.NoError(t, err)
// 	//
// 	mux.HandleFunc(pathHello1, func(resp http.ResponseWriter, req *http.Request) {
// 		slog.Info("TestAuthJWTFail: login")
// 		//_, _ = resp.Write([]byte("invalid token"))
// 		resp.WriteHeader(http.StatusUnauthorized)
// 	})
// 	//
// 	cl := tlsclient.NewTLSClient(testAddress, nil, authBundle.CaCert, 0)
// 	cl.ConnectWithToken(clientID, "badtoken")
// 	resp, _, err := cl.Post(pathHello1, []byte("test"))
// 	assert.Empty(t, resp)
// 	// unauthorized
// 	assert.Error(t, err)

// 	cl.Close()
// 	_ = srv.Close()
// }
